module Pages.Top exposing (Flags, Model, Msg, page)

import Data.Device as D
import Data.Sample exposing (Sample, renderSample)
import Dict exposing (Dict)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Global
import Html.Events
import Json.Decode as Decode
import Page exposing (Document, Page)
import Time
import UI.Form as Form
import UI.Icon as Icon
import UI.Style as Style exposing (colors, size)


type alias Flags =
    ()


type alias Model =
    { deviceEdits : Dict String String
    }


type alias DeviceEdit =
    { id : String
    , description : String
    }


type Msg
    = EditDeviceDescription DeviceEdit
    | PostConfig String D.Config
    | DiscardEditedDeviceDescription String
    | DeleteDevice String
    | Tick Time.Posix


page : Page Flags Model Msg
page =
    Page.component
        { init = init
        , update = update
        , subscriptions = subscriptions
        , view = view
        }


init : Global.Model -> Flags -> ( Model, Cmd Msg, Cmd Global.Msg )
init _ _ =
    ( Model Dict.empty, Cmd.none, Global.send Global.RequestDevices )


update : Global.Model -> Msg -> Model -> ( Model, Cmd Msg, Cmd Global.Msg )
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
            , Global.send <| Global.UpdateDeviceConfig id config
            )

        DiscardEditedDeviceDescription id ->
            ( { model | deviceEdits = Dict.remove id model.deviceEdits }
            , Cmd.none
            , Cmd.none
            )

        DeleteDevice id ->
            ( model, Cmd.none, Global.send <| Global.DeleteDevice id )

        Tick _ ->
            ( model
            , Cmd.none
            , Global.send Global.RequestDevices
            )


subscriptions : Global.Model -> Model -> Sub Msg
subscriptions _ _ =
    Sub.batch
        [ Time.every 1000 Tick
        ]


view : Global.Model -> Model -> Document Msg
view global model =
    { title = "Top"
    , body =
        [ column
            [ width fill, spacing 32 ]
            [ el Style.h2 <| text "Devices"
            , case global.auth of
                Global.SignedIn sess ->
                    viewDevices sess.data.devices model.deviceEdits

                _ ->
                    el [ padding 16 ] <| text "Sign in to view your devices."
            ]
        ]
    }


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
        , Border.color colors.black
        , spacing 6
        ]
        [ row []
            [ viewDeviceId device.id
            , Icon.x (DeleteDevice device.id)
            ]
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
                    , Font.color colors.gray
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
    , focused [ Background.color colors.yellow ]
    ]
        ++ (if modded then
                [ Background.color colors.orange
                , onEnter save
                , below <|
                    Form.buttonRow
                        [ Form.button "discard" colors.gray discard
                        , Form.button "save" colors.blue save
                        ]
                ]

            else
                [ Background.color colors.pale ]
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
