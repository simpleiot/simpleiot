module Pages.Orgs exposing (Model, Msg, page)

import Device
import Dict exposing (Dict)
import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Generated.Params as Params
import Global
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (hardcoded, optional, required)
import Org as O
import Spa.Page
import Spa.Types as Types
import Url.Builder as Url
import User
import Utils.Spa exposing (Page)
import Utils.Styles exposing (palette, size)


page : Page Params.Orgs Model Msg model msg appMsg
page =
    Spa.Page.element
        { title = always "Orgs"
        , init = init
        , update = always update
        , subscriptions = always subscriptions
        , view = always view
        }


page2 : Page Params.Orgs Model Msg model msg appMsg
page2 =
    Spa.Page.component
        { title = always "Orgs"
        , init = init2
        , update = always update2
        , subscriptions = always subscriptions
        , view = always view
        }



-- INIT


type alias Model =
    { orgs : List O.Org
    , error : Maybe Http.Error
    , emails : Dict String String
    }


init2 : Types.PageContext route Global.Model -> Params.Orgs -> ( Model, Cmd Msg, Cmd Global.Msg )
init2 context params =
    let
        ( model, msg ) =
            init context params
    in
    ( model
    , msg
    , Cmd.none
    )


init : Types.PageContext route Global.Model -> Params.Orgs -> ( Model, Cmd Msg )
init context _ =
    case context.global of
        Global.SignedIn sess ->
            ( empty
            , getOrgs sess.authToken
            )

        Global.SignedOut _ ->
            ( empty
            , Cmd.none
            )


empty =
    { orgs = []
    , error = Nothing
    , emails = Dict.empty
    }


-- UPDATE


type Msg
    = UpdateOrgs (Result Http.Error (List O.Org))
    | EditEmail String String


update2 : Msg -> Model -> ( Model, Cmd Msg, Cmd Global.Msg )
update2 msg model =
    let
        (model_, msg_) =
            update msg model
    in
    ( model_
    , msg_
    , Cmd.none
    )


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        UpdateOrgs (Ok orgs) ->
            ( { model | orgs = orgs }
            , Cmd.none
            )

        UpdateOrgs (Err err) ->
            ( { model | error = Just err }
            , Cmd.none
            )

        EditEmail id email ->
            ( { model | emails = Dict.insert id email model.emails }
            , Cmd.none
              -- TODO: does this user exist?
            )


getOrgs : String -> Cmd Msg
getOrgs token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "orgs" ] []
        , expect = Http.expectJson UpdateOrgs O.decodeList
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.none



-- VIEW


view : Model -> Element Msg
view model =
    column
        [ width fill, spacing 32 ]
        [ el [ padding 16, Font.size 24 ] <| text "Orgs"
        , viewError model.error
        , viewOrgs model
        ]


viewOrgs model =
    column
        [ width fill
        , spacing 40
        ]
    <|
        List.map (viewOrg model.emails) model.orgs


getEmail emails orgId =
    case Dict.get orgId emails of
        Just email ->
            email

        Nothing ->
            ""


viewOrg : Dict String String -> O.Org -> Element Msg
viewOrg emails org =
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color palette.black
        , spacing 6
        ]
        [ viewOrgName org.name
        , viewItems (getEmail emails org.id) org
        ]


viewItems email org =
    wrappedRow
        [ width fill
        , spacing 16
        ]
        [ viewUsers email org
        , viewDevices org.devices
        ]


viewUsers : String -> O.Org -> Element Msg
viewUsers email org =
    column
        []
        [ Input.text
            []
            { onChange = EditEmail org.id
            , text = email
            , placeholder = Nothing
            , label = label Input.labelAbove "Add user by email address"
            }
        , viewList "Users" viewUser org.users
        ]


label kind =
    kind
        [ padding 16
        , Font.italic
        , Font.color palette.gray
        ]
        << text


viewDevices =
    viewList "Devices" viewDevice


dup a =
    (++) a a


viewOrgName name =
    el
        [ padding 16
        , size.heading
        ]
    <|
        text name


viewList name fn list =
    column
        [ alignTop
        , width (fill |> minimum 250)
        , spacing 16
        ]
    <|
        [ el [ padding 16 ] <| text name ]
            ++ List.map fn list


viewItem =
    wrappedRow
        [ padding 16
        , spacing 25
        , Border.widthEach { top = 1, bottom = 0, left = 0, right = 0 }
        , Border.color palette.black
        , width fill
        ]


viewUser : User.User -> Element Msg
viewUser user =
    viewItem
        [ text user.first
        , text user.last
        ]


viewDevice : Device.Device -> Element Msg
viewDevice device =
    viewItem
        [ text device.id
        , text device.config.description
        ]


viewError error =
    case error of
        Just (Http.BadUrl str) ->
            text <| "bad URL: " ++ str

        Just Http.Timeout ->
            text "timeout"

        Just Http.NetworkError ->
            text "network error"

        Just (Http.BadStatus status) ->
            text <| "bad status: " ++ String.fromInt status

        Just (Http.BadBody str) ->
            text <| "bad body: " ++ str

        Nothing ->
            none
