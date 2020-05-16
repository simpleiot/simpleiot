module Pages.Devices exposing
    ( Model
    , Msg
    , Response
    , page
    )

import Components.Form as Form
import Data.Device as D
import Data.Sample exposing (Sample, renderSample)
import Dict exposing (Dict)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Generated.Params as Params
import Global
import Html.Events
import Json.Decode as Decode
import Spa.Page
import Spa.Types as Types
import Time
import Utils.Spa exposing (Page)
import Utils.Styles exposing (palette, size)


page : Page Params.Devices Model Msg model msg appMsg
page =
    Spa.Page.component
        { title = always "Devices"
        , init = always init
        , update = update
        , subscriptions = subscriptions
        , view = view
        }



-- INIT


type alias Model =
    { deviceEdits : Dict String String
    }


init : Params.Devices -> ( Model, Cmd Msg, Cmd Global.Msg )
init _ =
    ( { deviceEdits = Dict.empty
      }
    , Cmd.none
    , Spa.Page.send Global.RequestDevices
    )



-- UPDATE


type Msg
    = EditDeviceDescription DeviceEdit
    | PostConfig String D.Config
    | DiscardEditedDeviceDescription String
    | Tick Time.Posix


type alias Response =
    { success : Bool
    , error : String
    , id : String
    }


update : Types.PageContext route Global.Model -> Msg -> Model -> ( Model, Cmd Msg, Cmd Global.Msg )
update _ msg model =
    case msg of
        EditDeviceDescription { id, description } ->
            ( { model | deviceEdits = Dict.insert id description model.deviceEdits }
            , Cmd.none
            , Cmd.none
            )

        PostConfig id config ->
            ( { model | deviceEdits = Dict.remove id model.deviceEdits }
            , Cmd.none
            , Spa.Page.send <| Global.UpdateDeviceConfig id config
            )

        DiscardEditedDeviceDescription id ->
            ( { model | deviceEdits = Dict.remove id model.deviceEdits }
            , Cmd.none
            , Cmd.none
            )

        Tick _ ->
            ( model
            , Cmd.none
            , Spa.Page.send Global.RequestDevices
            )


type alias DeviceEdit =
    { id : String
    , description : String
    }



-- SUBSCRIPTIONS


subscriptions : Types.PageContext route Global.Model -> Model -> Sub Msg
subscriptions _ _ =
    Sub.batch
        [ Time.every 1000 Tick
        ]



-- VIEW


view : Types.PageContext route Global.Model -> Model -> Element Msg
view context model =
    column
        [ width fill, spacing 32 ]
        [ el [ padding 16, Font.size 24 ] <| text "Devices"
        , case context.global of
            Global.SignedIn sess ->
                viewDevices sess.data.devices model.deviceEdits

            _ ->
                el [ padding 16 ] <| text "Sign in to view your devices."
        ]


viewDevices : List D.Device -> Dict String String -> Element Msg
viewDevices devices edits =
    column
        [ width fill
        , spacing 24
        ]
    <|
        List.map (viewDevice edits) devices


viewDevice : Dict String String -> D.Device -> Element Msg
viewDevice edits device =
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color palette.black
        , spacing 6
        ]
        [ viewDeviceId device.id
        , viewDeviceDescription edits device
        , viewIoList device.state.ios
        ]


viewDeviceDescription : Dict String String -> D.Device -> Element Msg
viewDeviceDescription edits device =
    descriptionField
        device.id
        { description = deviceDescription edits device }
        (modified edits device)


viewDeviceId : String -> Element Msg
viewDeviceId id =
    el
        [ padding 16
        , size.heading
        ]
    <|
        text id


viewIoList : List Sample -> Element Msg
viewIoList ios =
    column
        [ padding 16
        , spacing 6
        ]
    <|
        List.map (renderSample >> text) ios


deviceDescription : Dict String String -> D.Device -> String
deviceDescription edits device =
    case Dict.get device.id edits of
        Just desc ->
            desc

        Nothing ->
            device.config.description


modified : Dict String String -> D.Device -> Bool
modified edits device =
    case Dict.get device.id edits of
        Just desc ->
            desc /= device.config.description

        Nothing ->
            False


descriptionField : String -> D.Config -> Bool -> Element Msg
descriptionField id config modded =
    Input.text
        (fieldAttrs
            modded
            (PostConfig id config)
            (DiscardEditedDeviceDescription id)
        )
        { onChange =
            \d ->
                EditDeviceDescription
                    { id = id
                    , description = d
                    }
        , text = config.description
        , placeholder =
            Just <|
                Input.placeholder
                    [ Font.italic
                    , Font.color palette.gray
                    ]
                <|
                    text "description"
        , label = Input.labelHidden "Description"
        }


fieldAttrs : Bool -> Msg -> Msg -> List (Attribute Msg)
fieldAttrs modded save discard =
    [ padding 16
    , width fill
    , Border.width 0
    , Border.rounded 0
    , focused [ Background.color palette.yellow ]
    ]
        ++ (if modded then
                [ Background.color palette.orange
                , onEnter save
                , below <|
                    Form.buttonRow
                        [ Form.button "discard" palette.pale discard
                        , Form.button "save" palette.green save
                        ]
                ]

            else
                [ Background.color palette.pale ]
           )


onEnter : msg -> Attribute msg
onEnter msg =
    htmlAttribute
        (Html.Events.on "keyup"
            (Decode.field "key" Decode.string
                |> Decode.andThen
                    (\key ->
                        if key == "Enter" then
                            Decode.succeed msg

                        else
                            Decode.fail "Not the enter key"
                    )
            )
        )
